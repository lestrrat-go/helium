<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

    <!-- Test the document-uri() function -->



<xsl:variable name="x"><x/></xsl:variable>

<xsl:template match="/" xml:base="http://no.other/home/">
  <out>
    <a><xsl:value-of select="substring-after(document-uri(/), '/fn/')"/></a>
    <a><xsl:value-of select="document-uri($x)"/></a>              
  </out>
</xsl:template>

</xsl:stylesheet>
