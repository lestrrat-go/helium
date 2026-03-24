<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: To be imported by ../ImpIncl17. -->

<xsl:key name="id" use="@id" match="LAMBDA"/>
<xsl:key name="annid" use="@of" match="Annotation"/>

<xsl:template match="a">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="Definition"/><!-- Don't handle the LAMBDAs now -->

<!-- Alternate template for Definition...
Usual "apply" looping, which means we will visit the LAMBDAs -->
<!--
<xsl:template match="Definition">
  <xsl:apply-templates/>
</xsl:template>
-->

<xsl:template match="LAMBDA">
  <xsl:choose>
    <xsl:when test="key('annid',@id)">
      <xsl:text>NO BUG</xsl:text>
    </xsl:when>
    <xsl:otherwise>
      <xsl:text>Found one whose id has no annotation!</xsl:text>
    </xsl:otherwise>
  </xsl:choose>
</xsl:template>

<xsl:template match="node">
  <xsl:variable name="id" select="@id"/>
  <xsl:text>On node whose id is </xsl:text>
  <xsl:value-of select="$id"/>
  <xsl:text> -nodes to apply: </xsl:text>
  <xsl:value-of select="count(key('id',$id))"/><xsl:text>
</xsl:text>
  <xsl:apply-templates select="key('id',$id)"/>
</xsl:template>

</xsl:stylesheet>
