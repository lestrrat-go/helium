<?spec xslt#grouping?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- tests current(), position() and last() within xsl:for-each-group -->


  <xsl:variable name="endline" select="'&#x0a;'"/>

  <xsl:template match="/">
    <z>
      <xsl:for-each-group select="CASTDataset/ETL_AIR_7" group-by="@CARRIER_NM">
        <xsl:value-of select="concat('The current node is ', current()/@ACFT_TYP_SYN_NM)"/>
        <xsl:value-of select="$endline"/>
        <xsl:value-of
          select="concat(' For the ', @CARRIER_NM, 
                             ' group, the current position is ', position(), ' and last is ',
                               last())"/>
        <xsl:value-of select="$endline"/>
        <xsl:for-each select="current-group()">
          <xsl:value-of
            select="concat('For the items within the current-group(), ACFT_TYP_SYN_NM is ',
                              @ACFT_TYP_SYN_NM, ',the current position is ',
                              position(), ' and last is ', last())"/>
          <xsl:value-of select="$endline"/>
        </xsl:for-each>
      </xsl:for-each-group>
    </z>
  </xsl:template>

</xsl:stylesheet>
