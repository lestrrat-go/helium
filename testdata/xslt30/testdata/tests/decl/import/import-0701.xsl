<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:include href="q.xsl"/>

<?spec xslt#include?>
    <!-- Purpose: The resource located by the href attribute value is parsed as an 
       XML document, and the children of the xsl:stylesheet element in this document 
       replace the xsl:include element in the including document. (No namespaces 
       should be copied over. -->

<xsl:template match="/">
  <out>
    <xsl:element name="test" use-attribute-sets="set1 set2" namespace="http://www.lotus.com/"/>
  </out>
</xsl:template>

<xsl:attribute-set name="set2">
  <xsl:attribute name="text-decoration">underline</xsl:attribute>
  <xsl:attribute name="test" ></xsl:attribute>
</xsl:attribute-set>

<xsl:attribute-set name="set1">
  <xsl:attribute name="color">black</xsl:attribute>
</xsl:attribute-set>

</xsl:stylesheet>
