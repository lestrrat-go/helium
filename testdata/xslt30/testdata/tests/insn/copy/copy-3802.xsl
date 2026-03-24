<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Test deep-equal on the result of a copy including comments, whitespace, etc -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>
  
  <!--<xsl:variable name="deep-equal" static="yes" xmlns:saxon="http://saxon.sf.net/"
    select="if (function-available('saxon:deep-equal', 4)) then saxon:deep-equal(?, ?, (), '?') else deep-equal#2"/>-->

<xsl:template match="/">
  <xsl:variable name="temp">
    <z1>
      <z2>
        <z3>
          <xsl:copy-of select="."/>
        </z3>
        <z4>
          <xsl:copy-of select="./*"/>
        </z4>
      </z2>
    </z1>
  </xsl:variable>
  <out>
    <z><xsl:copy-of select="$temp"/></z>
    <z><xsl:copy-of select="."/></z>
    <a><xsl:value-of select="deep-equal($temp//z3/*, ./*)"/></a>
    <b><xsl:value-of select="deep-equal($temp//z4/*, ./*)"/></b>
  </out>
</xsl:template>

</xsl:stylesheet>