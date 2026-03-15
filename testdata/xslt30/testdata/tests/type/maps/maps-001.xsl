<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local-functions.org/"
    exclude-result-prefixes="map xs f">
    
    <xsl:variable name="sales" as="map(*)*" select='
        map{ "product" : "broiler", "store number" : 1, "quantity" : 20  },
        map{ "product" : "toaster", "store number" : 2, "quantity" : 100 },
        map{ "product" : "toaster", "store number" : 2, "quantity" : 50 },
        map{ "product" : "toaster", "store number" : 3, "quantity" : 50 },
        map{ "product" : "blender", "store number" : 3, "quantity" : 100 },
        map{ "product" : "blender", "store number" : 3, "quantity" : 150 },
        map{ "product" : "socks", "store number" : 1, "quantity" : 500 },
        map{ "product" : "socks", "store number" : 2, "quantity" : 10 },
        map{ "product" : "shirt", "store number" : 3, "quantity" : 10 }'/>
    
    <xsl:variable name="products" as="map(*)*" select='
        map{ "name" : "broiler", "category" : "kitchen", "price" : 100, "cost" : 70 },
        map{ "name" : "toaster", "category" : "kitchen", "price" : 30, "cost" : 10 },
        map{ "name" : "blender", "category" : "kitchen", "price" : 50, "cost" : 25 },
        map{ "name" : "socks", "category" : "clothes", "price" : 5, "cost" : 2 },
        map{ "name" : "shirt", "category" : "clothes", "price" : 10, "cost" : 3 }'/>
    
    <xsl:variable name="stores" as="map(*)*" select='
        map{ "store number" : 1, "state" : "CA" },
        map{ "store number" : 2, "state" : "CA" },
        map{ "store number" : 3, "state" : "MA" },
        map{ "store number" : 4, "state" : "MA" }'/>
    
    <xsl:template name="main">
        <xsl:variable name="state-maps" as="map(*)*">
            <xsl:for-each-group select="$stores" group-by=".('state')">
                <xsl:variable name="state" select="current-grouping-key()" as="xs:string"/>
                <xsl:variable name="stores-in-state" select="current-group()!.('store number')" as="xs:integer*"/>
                <xsl:variable name="state-map-entry" as="map(*)*">
                    <xsl:for-each-group select="$products" group-by=".('category')">
                        <xsl:variable name="category" select="current-grouping-key()" as="xs:string"/>
                        <xsl:variable name="products-in-category" select="current-group()" as="map(*)*"/>
                        <xsl:variable name="totals-map" as="map(*)*">
                            <xsl:variable name="totals-map-entries" as="map(*)*">
                                <xsl:for-each select="$products-in-category">
                                   <xsl:variable name="product-name" select=".('name')"/>
                                   <xsl:variable name="product-sales" 
                                       select="$sales[.('product') = $product-name and 
                                                         .('store number') = $stores-in-state]"/>                      
                                   <xsl:if test="exists($product-sales)">                      
                                      <xsl:sequence select="map{ $product-name : sum($product-sales!.('quantity')) }"/>
                                   </xsl:if>   
                                </xsl:for-each>
                            </xsl:variable>
                            <xsl:sequence select="map:merge($totals-map-entries)"/>
                        </xsl:variable>
                        <xsl:sequence select="map{ $category : $totals-map }"/>
                    </xsl:for-each-group>
                </xsl:variable>    
                <xsl:sequence select=" map { $state : $state-map-entry }"/>
            </xsl:for-each-group>
        </xsl:variable>
        <json>
          <xsl:text>[</xsl:text>
            <xsl:for-each select="$state-maps">
              <xsl:if test="position() ne 1">,</xsl:if>
              <xsl:apply-templates select="." mode="to-json"/>
            </xsl:for-each>
          <xsl:text>]</xsl:text>
        </json>  
    </xsl:template>
    
    <xsl:template match=".[. instance of map(*)]" mode="to-json">
      <xsl:variable name="m" select="." as="map(*)"/>
      <xsl:text>{</xsl:text>
      <xsl:for-each select="map:keys($m)">
        <xsl:sort select="."/>
        <xsl:if test="position() ne 1">,</xsl:if>
        <xsl:text>"</xsl:text>
        <xsl:value-of select="."/>
        <xsl:text>":</xsl:text>
        <xsl:choose>
          <xsl:when test="count($m(.)) ne 1">
            <xsl:text>[</xsl:text>
            <xsl:for-each select="$m(.)">
              <xsl:if test="position() ne 1">,</xsl:if>
              <xsl:apply-templates select="." mode="to-json"/>
            </xsl:for-each>
            <xsl:text>]</xsl:text>
          </xsl:when>
          <xsl:otherwise>
            <xsl:apply-templates select="$m(.)" mode="to-json"/>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:for-each>
      <xsl:text>}</xsl:text>
    </xsl:template>
    
    <xsl:template match=".[. instance of xs:double]" mode="to-json">
      <xsl:value-of select="."/>
    </xsl:template>
    
    <xsl:template match=".[. instance of xs:boolean]" mode="to-json">
      <xsl:value-of select="."/>
    </xsl:template>  
              
    <xsl:template match="." mode="to-json">
      <xsl:text>"</xsl:text>
      <xsl:value-of select="."/>
      <xsl:text>"</xsl:text>
    </xsl:template>  
          

    
</xsl:stylesheet>