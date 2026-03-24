<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs" expand-text="yes">
  <!-- Purpose: match="$var" where $var is heterogeneous -->

  <xsl:variable name="e" as="element()"><e/></xsl:variable>
  <xsl:param name="var" select="($e, $e, 1234)"/>

  <xsl:template name="xsl:initial-template">
    <xsl:apply-templates select="$e"/>
  </xsl:template>

  <xsl:template match="$var" priority="100">
    <!-- This is an error, because //node()/$var is an error; but the processor
         can legitimately match the node before detecting the error. If it does
         detect the error, it must not report it, but must treat it as a non-match -->
    <ok option="1"/>
  </xsl:template>

  <xsl:template match="*" priority="20">
    <ok option="2"/>
  </xsl:template>

</xsl:transform>
