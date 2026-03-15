<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#attribute-value-templates?>
    <!-- Purpose: Double braces to neutralize AVT processing, including colon. -->

<xsl:template match="doc">
  <out style="{{font:helvetica}}"/>
</xsl:template>

</xsl:stylesheet>