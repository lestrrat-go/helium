<xsl:package version="3.0" 
  xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  exclude-result-prefixes="xs"  declared-modes="no">

<xsl:accumulator name="firstTitle" as="xs:string?" initial-value="()">
  <xsl:accumulator-rule match="/html/head/title[1]/text()" select="string(.)"/>
</xsl:accumulator>
  
<xsl:mode use-accumulators="firstTitle"/>  

<xsl:template match="@* | node()">
  <xsl:copy>
    <xsl:copy-of select="@*"/>
    <xsl:apply-templates/>
  </xsl:copy>
</xsl:template>

<xsl:template match="h1">
  <xsl:variable name="firstTitle" select="accumulator-before('firstTitle')"/>
  <xsl:variable name="thisTitle" select="string(.)"/>
  <xsl:if test="$thisTitle ne $firstTitle">
    <div class="heading-1"><xsl:value-of select="$thisTitle"/></div>
  </xsl:if>
</xsl:template>

</xsl:package>
