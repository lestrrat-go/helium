<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#value-of?>
    <!-- Purpose: Try to put out computed text without any tags.-->

<xsl:output method="text"/>

<xsl:template match="doc">
  <xsl:value-of select="inner"/>
</xsl:template>

</xsl:stylesheet>
