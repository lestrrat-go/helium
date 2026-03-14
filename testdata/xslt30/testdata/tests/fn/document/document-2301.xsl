<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:months="http://ns.uri/Lookup-table-for-month-names"
                exclude-result-prefixes="months">
  <!-- Purpose: Use document('') to refer to the stylesheet, and have a local lookup table. -->

  <months:name sequence="01">January</months:name>
  <months:name sequence="02">February</months:name>
  <months:name sequence="03">March</months:name>
  <months:name sequence="04">April</months:name>
  <months:name sequence="05">May</months:name>
  <months:name sequence="06">June</months:name>
  <months:name sequence="07">July</months:name>
  <months:name sequence="08">August</months:name>
  <months:name sequence="09">September</months:name>
  <months:name sequence="10">October</months:name>
  <months:name sequence="11">November</months:name>
  <months:name sequence="12">December</months:name>


<xsl:template match="/">
  <out>
    <xsl:for-each select="/report/month">
      <month>
        <xsl:value-of select="document('')/xsl:stylesheet/months:name[@sequence=current()/@sequence]"/>
        <xsl:text> - </xsl:text>
        <xsl:value-of select="miles-earned"/>
        <xsl:text> miles earned.</xsl:text>
      </month>
    </xsl:for-each>
  </out>
</xsl:template>

</xsl:stylesheet>