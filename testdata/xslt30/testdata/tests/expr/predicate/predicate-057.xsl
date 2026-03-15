<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet exclude-result-prefixes="#all" version="3.0"
   xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
   
   <xsl:template name="xsl:initial-template">
      <i>
         <xsl:value-of select="('0', 0)[1] ! xs:integer(.)" />
      </i>
   </xsl:template>
   
</xsl:stylesheet>