<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:target="http://name-b.com/"
    exclude-result-prefixes="target"
    version="2.0">

  <!-- FileName: copy51 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 7.5 -->
    <!-- Purpose: Test for xsl:copy on namespaces referenced directly. -->
  <!-- Elaboration: while namespace::* will include the implied declaration for
    The XML Namespace, it's not appropriate to serialize it as an explicit declaration.
    XML output from here could be pipelined to some other process that wants the XML
    namespace to be implicit only. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="/">
  <out>
    <xsl:apply-templates select=".//target:sub"/>
  </out>
</xsl:template>

<xsl:template match="target:sub">
  <xsl:for-each select="namespace::*">
    <xsl:sort select="local-name(.)"/>
    <xsl:copy/>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>