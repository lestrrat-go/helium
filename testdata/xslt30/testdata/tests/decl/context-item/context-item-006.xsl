<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with required context item, but none is supplied -->
  
    <xsl:template name="t">
      <xsl:context-item use="required"/>
      <xsl:sequence select="."/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
        <xsl:call-template name="t"/>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>