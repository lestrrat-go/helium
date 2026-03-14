<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs" version="3.0"  declared-modes="no">

  <!-- Use an accumulator within a global variable -->
  
  <xsl:param name="streamable" static="yes" required="yes"/>
  
  <xsl:mode _streamable="{$streamable}"/>
  
  <xsl:accumulator name="a" as="xs:integer" initial-value="0" streamable="yes">
    <xsl:accumulator-rule match="*" select="$value + 1"/>
  </xsl:accumulator>

  <xsl:variable name="count" use-when="$streamable='yes'">
    <xsl:source-document streamable="true" href="accumulator-001.xml">
      <!--<xsl:for-each select="*"/>-->
      <xsl:value-of select="accumulator-after('a')"/>
    </xsl:source-document>
  </xsl:variable>
  
  <xsl:variable name="count" use-when="$streamable='no'">
    <xsl:for-each select="doc('accumulator-001.xml')">
      <!--<xsl:for-each select="*"/>-->
      <xsl:value-of select="accumulator-after('a')"/>
    </xsl:for-each>
  </xsl:variable>
  
  <xsl:template name="main" visibility="public">
    <o><xsl:value-of select="$count"/></o>
  </xsl:template>
</xsl:package>
