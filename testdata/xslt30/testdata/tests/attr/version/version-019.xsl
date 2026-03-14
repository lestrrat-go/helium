<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="x xs"
xmlns:x="http://x.uri/">

<!-- PURPOSE:  Test extension attributes on various elements -->
<?spec xslt#extension-attributes?>

<xsl:preserve-space elements="a b c" x:att="12.0"/>

<xsl:template match="doc" x:att="1.5">
<out xsl:version="13.9">
  <xsl:text x:att="3">Hello</xsl:text>
</out>
</xsl:template>
</xsl:stylesheet>