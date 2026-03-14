<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="#all"
   expand-text="yes">
   
   
   <xsl:mode on-no-match="shallow-copy" streamable="yes"/>
   
   <xsl:template match="/">
      <xsl:try>
         <xsl:apply-templates/>
         <xsl:catch/>
      </xsl:try>
   </xsl:template>
   
</xsl:stylesheet>