<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
    <!-- Purpose: a trivial predicate can affect the pattern priority -->

<xsl:template match="doc[true()]">
 <out>doc[true()]</out>
</xsl:template>

<xsl:template match="doc">
 <out>doc</out>
</xsl:template>

</xsl:stylesheet>

