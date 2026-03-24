<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs"
    version="3.0">

    
    <xsl:template name="main" expand-text="yes">
      <out>
       <xsl:for-each select="1 to 3">
         <n>{(3 to 5)[. = current()]}</n>
       </xsl:for-each>
      </out>   
    </xsl:template>
    
</xsl:stylesheet>