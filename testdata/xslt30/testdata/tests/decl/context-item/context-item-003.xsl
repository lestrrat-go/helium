<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with required type element(x); supplied value is correct type -->
  
    <xsl:template name="t">
      <xsl:context-item as="(:1:) element (:2:) ( (:3:) doc (:4:) ) (:5:)"/>
      <xsl:sequence select="."/>
    </xsl:template>
    
    <xsl:template match="/doc">
      <out>
        <xsl:call-template name="t"/>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>