<?xml version="1.0" encoding="utf-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
 xmlns:xs="http://www.w3.org/2001/XMLSchema-datatypes" version="3.0" expand-text="yes">

 <xsl:template name="xsl:initial-template">
  <out>{(160 to 256) => codepoints-to-string() => lower-case() => string-to-codepoints()}</out>
 </xsl:template>


</xsl:stylesheet>