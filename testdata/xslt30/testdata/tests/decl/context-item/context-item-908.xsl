<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:f="http://local-functions/" exclude-result-prefixes="xs f">

    <!-- Misplaced xsl:context-item -->
  
    <xsl:template name="f:t">
      banana
      <xsl:context-item use="absent"/>
      <xsl:sequence select="22"/>
    </xsl:template>
    
    <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:call-template name="f:t"/>
         </xsl:for-each>
      </out>
    </xsl:template>  
          
  </xsl:stylesheet>