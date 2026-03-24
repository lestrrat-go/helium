<?spec xslt#grouping?>
<xsl:transform version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <!-- use case for grouping to merge an update file with a base file -->

  <xsl:template match="/">
    <out>
      <xsl:variable name="update-file" select="document('group015b.xml')"/>
      <xsl:for-each-group select="/*/employee, $update-file/*/employee" group-by="@SSN">
        <xsl:choose>
          <xsl:when test="count(current-group())=1">
            <xsl:copy-of select="current-group()"/>
          </xsl:when>
          <xsl:otherwise>
            <xsl:copy-of select="current-group()[ancestor::node() intersect $update-file]"/>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:for-each-group>
    </out>
  </xsl:template>

</xsl:transform>
