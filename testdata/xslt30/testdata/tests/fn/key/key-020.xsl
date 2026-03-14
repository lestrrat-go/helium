<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <?spec xslt#keys?>
  <!-- Purpose: Test for three keyspaces, some being imported. -->

  <xsl:import href="key-020a.xsl"/>

  <xsl:key name="filterspace" match="div[@allow='yes']" use="subdiv/title"/>

  <xsl:template match="doc">
    <root>
      <xsl:text>
bigspace...</xsl:text>
      <xsl:value-of select="key('bigspace', 'Introduction')/subdiv/p"/>
      <xsl:value-of select="key('bigspace', 'Stylesheet Structure')/subdiv/p"/>
      <xsl:value-of select="key('bigspace', 'Expressions')/subdiv/p"/>
      <xsl:text>
smallspace...</xsl:text>
      <xsl:value-of select="key('smallspace', 'Introduction')/p"/>
      <xsl:value-of select="key('smallspace', 'Stylesheet Structure')/p"/>
      <xsl:value-of select="key('smallspace', 'Expressions')/p"/>
      <xsl:text>
filterspace...</xsl:text>
      <xsl:value-of select="key('filterspace', 'Introduction')/subdiv/p"/>
      <xsl:value-of select="key('filterspace', 'Stylesheet Structure')/subdiv/p"/>
      <xsl:value-of select="key('filterspace', 'Expressions')/subdiv/p"/>
    </root>
  </xsl:template>

</xsl:stylesheet>
