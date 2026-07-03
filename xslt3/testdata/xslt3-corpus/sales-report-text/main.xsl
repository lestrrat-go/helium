<?xml version="1.0" encoding="UTF-8"?>
<!-- Entry module for the sales-report-text case.
     Exercises: xsl:include (keys.xsl), xsl:key + key(), xsl:for-each-group
     (group-by), xsl:sort, several params, and text output. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    exclude-result-prefixes="xs">

  <xsl:include href="keys.xsl"/>

  <xsl:output method="text" encoding="UTF-8"/>

  <xsl:param name="reportTitle" as="xs:string" select="'Quarterly Sales'"/>
  <xsl:param name="currencySymbol" as="xs:string" select="'$'"/>
  <xsl:param name="minTotal" as="xs:double" select="0"/>

  <xsl:template match="/sales">
    <xsl:value-of select="concat('Sales Report: ', $reportTitle, '&#10;')"/>
    <xsl:text>========================================&#10;</xsl:text>
    <xsl:for-each-group select="sale" group-by="@region">
      <xsl:sort select="current-grouping-key()"/>
      <xsl:variable name="total" select="sum(current-group()/@amount)"/>
      <xsl:if test="$total ge $minTotal">
        <xsl:variable name="name" as="xs:string"
            select="string(key('region-by-code', current-grouping-key())/@name)"/>
        <xsl:value-of select="concat($name, ' (', current-grouping-key(), '): ',
            count(current-group()), ' sales, total ', $currencySymbol,
            format-number($total, '#,##0.00'), '&#10;')"/>
      </xsl:if>
    </xsl:for-each-group>
    <xsl:text>----------------------------------------&#10;</xsl:text>
    <xsl:value-of select="concat('Grand total: ', $currencySymbol,
        format-number(sum(sale/@amount), '#,##0.00'), '&#10;')"/>
  </xsl:template>

</xsl:stylesheet>
