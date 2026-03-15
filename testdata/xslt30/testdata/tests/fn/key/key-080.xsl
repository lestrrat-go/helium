<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:kn="http://www.w3.org/1066/key-namespace">

<?spec xslt#xsl-key?>
<!-- test use of an invalid key name known only at run time -->

<xsl:key name="k" match="b" use="."/>
<xsl:param name="p" select="'kn:k'"/>

<xsl:template match="/">
  <xsl:sequence select="key($p, '42')"/>
  <a>** error not detected **</a>
</xsl:template>

</xsl:stylesheet>
