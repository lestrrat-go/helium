<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="map xs">
    
    <xsl:import href="maps-001.xsl"/>
    
    <xsl:output method="xml"/>
    
    <xsl:variable name="sales" as="map(*)*">
      <xsl:map>
        <xsl:map-entry key="'product'" select="'broiler'"/>
        <xsl:map-entry key="'store number'" select="1"/>
        <xsl:map-entry key="'quantity'" select="20"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'toaster'"/>
        <xsl:map-entry key="'store number'" select="2"/>
        <xsl:map-entry key="'quantity'" select="100"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'toaster'"/>
        <xsl:map-entry key="'store number'" select="2"/>
        <xsl:map-entry key="'quantity'" select="50"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'toaster'"/>
        <xsl:map-entry key="'store number'" select="3"/>
        <xsl:map-entry key="'quantity'" select="50"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'blender'"/>
        <xsl:map-entry key="'store number'" select="3"/>
        <xsl:map-entry key="'quantity'" select="100"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'blender'"/>
        <xsl:map-entry key="'store number'" select="3"/>
        <xsl:map-entry key="'quantity'" select="150"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'socks'"/>
        <xsl:map-entry key="'store number'" select="1"/>
        <xsl:map-entry key="'quantity'" select="500"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'socks'"/>
        <xsl:map-entry key="'store number'" select="2"/>
        <xsl:map-entry key="'quantity'" select="10"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'product'" select="'shirt'"/>
        <xsl:map-entry key="'store number'" select="3"/>
        <xsl:map-entry key="'quantity'" select="10"/>
      </xsl:map>
    </xsl:variable>
    
    <xsl:variable name="products" as="map(*)*">
      <xsl:map>
        <xsl:map-entry key="'name'" select="'broiler'"/>
        <xsl:map-entry key="'category'" select="'kitchen'"/>
        <xsl:map-entry key="'price'" select="100"/>
        <xsl:map-entry key="'cost'" select="70"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'name'" select="'toaster'"/>
        <xsl:map-entry key="'category'" select="'kitchen'"/>
        <xsl:map-entry key="'price'" select="30"/>
        <xsl:map-entry key="'cost'" select="10"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'name'" select="'blender'"/>
        <xsl:map-entry key="'category'" select="'kitchen'"/>
        <xsl:map-entry key="'price'" select="50"/>
        <xsl:map-entry key="'cost'" select="25"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'name'" select="'socks'"/>
        <xsl:map-entry key="'category'" select="'clothes'"/>
        <xsl:map-entry key="'price'" select="5"/>
        <xsl:map-entry key="'cost'" select="2"/>
      </xsl:map>
      <xsl:map>
        <xsl:map-entry key="'name'" select="'shirt'"/>
        <xsl:map-entry key="'category'" select="'clothes'"/>
        <xsl:map-entry key="'price'" select="10"/>
        <xsl:map-entry key="'cost'" select="3"/>
      </xsl:map>
    </xsl:variable>
    
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
    
</xsl:stylesheet>