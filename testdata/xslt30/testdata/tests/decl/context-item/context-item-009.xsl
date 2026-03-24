<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with use="optional" defaulted-->
  
    <xsl:template name="t">
      <xsl:context-item as="xs:integer"/>
      <xsl:param name="p" select="3"/>
      <xsl:sequence select="if ($p = 3) then . else $p"/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
        <xsl:for-each select="1 to 5">
          <xsl:call-template name="t"/>
        </xsl:for-each> 
        <xsl:call-template name="t">
          <xsl:with-param name="p" select="6"/>
        </xsl:call-template> 
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>