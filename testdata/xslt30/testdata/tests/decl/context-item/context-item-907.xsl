<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:f="http://local-functions/" exclude-result-prefixes="xs f">

    <!-- Misplaced xsl:context-item -->
  
    <xsl:function name="f:t">
      <xsl:context-item use="absent"/>
      <xsl:sequence select="22"/>
    </xsl:function>
    
    <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:sequence select="f:t()"/>
         </xsl:for-each>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>