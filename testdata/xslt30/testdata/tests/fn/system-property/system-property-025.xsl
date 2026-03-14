<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   exclude-result-prefixes="#all"
   expand-text="yes">
   
   <!-- Purpose: system-property() called in a TVT with a locally-declared namespace prefix -->

   <xsl:template name="xsl:initial-template">
      <out>
         <xsl:text xmlns:zz="http://www.w3.org/1999/XSL/Transform">Run with {system-property('zz:version')}</xsl:text>
      </out>
   </xsl:template>
   
</xsl:stylesheet>