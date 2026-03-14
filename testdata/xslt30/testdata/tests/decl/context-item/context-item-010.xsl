<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Test xsl:context-item with use="absent" and a required type.-->
    <!-- Spec unclear see bug 27014 -->
  
    <xsl:template name="t">
      <xsl:context-item as="xs:integer" use="absent"/>
      <xsl:param name="p" select="3"/>
      <xsl:sequence select="if ($p = 3) then . else $p"/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
        <xsl:call-template name="t">
          <xsl:with-param name="p" select="6"/>
        </xsl:call-template> 
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>