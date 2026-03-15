<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema"
xmlns:kn="http://www.w3.org/1066/key-namespace">

<?spec xslt#xsl-key?>
<!-- test use of an invalid key name known at compile time -->

<xsl:key name="kn:k" match="b" use="."/>

<xsl:template match="/">
  <xsl:sequence select="key('k', '42')"/>
  <a>** error not detected **</a>
</xsl:template>

</xsl:stylesheet>
