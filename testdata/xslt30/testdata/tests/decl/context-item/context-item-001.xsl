<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with required type; supplied value matches -->
  
    <xsl:template name="t">
      <xsl:context-item as="xs:integer"/>
      <xsl:sequence select="."/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:call-template name="t"/>
         </xsl:for-each>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>