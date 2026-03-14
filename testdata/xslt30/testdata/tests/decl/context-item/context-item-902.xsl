<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

    <!-- Unknown type in xsl:context-item/@as -->
  
    <xsl:template name="t">
      <xsl:context-item as="my:percentage" xmlns:my="http://all.mine.com/"/>
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