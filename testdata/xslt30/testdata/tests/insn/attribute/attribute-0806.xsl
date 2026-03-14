<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: XSLT processors may make use of the prefix of the QNAME specified
       in the name attribute ... however they are not required to do so and, if 
       the prefix is xmlns, they must not do so ... this will not result in a 
       namespace declaration being output. -->

<xsl:template match="/">
 <root>
   <!-- the test produces an attribute with an implementation-defined namespace prefix. We avoid
        outputting this directly to ensure repeatable results -->
  <xsl:variable name="out" as="element()">      
  <Out>

	<xsl:attribute name="xmlns:xsl" namespace="http://whatever.example.com/">http://www.w3.org/1999/XSL/Transform</xsl:attribute>
  </Out>
  </xsl:variable>
  <lnames><xsl:value-of select="$out/@*/local-name()"/></lnames>
  <uris><xsl:value-of select="$out/@*/namespace-uri()"/></uris>
  <xsl:for-each select="$out//namespace::*">
    <xsl:sort select="."/>
    <inscope><xsl:value-of select="."/></inscope>

  </xsl:for-each>
 </root>
</xsl:template>

</xsl:stylesheet>


