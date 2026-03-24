<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test with declared return type, error case -->
  
    <xsl:template name="t" as="xs:string">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:call-template name="t-private">
         <xsl:with-param name="in" select="$in"/>
         <xsl:with-param name="count" select="$count"/>
      </xsl:call-template>
    </xsl:template>
    
    <xsl:template name="t-private" as="xs:string*">
      <xsl:param name="in" as="xs:string"/>
      <xsl:param name="count" as="xs:integer"/>
      <xsl:sequence select="(1 to $count)!$in"/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
         <xsl:call-template name="t">
            <xsl:with-param name="in" select="'x'"/>
            <xsl:with-param name="count" select="5"/>
         </xsl:call-template>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>