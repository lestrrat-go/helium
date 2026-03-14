<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
  xmlns:local="http://copy903err/ns">

  <!-- FileName: copy903err -->
    <!-- Spec: XSLT 2.0 section 11.9.1 -->
  <!-- Purpose: Using xsl:copy when there is no context item is an error. -->
  <!--    In this case the error can be detected statically -->
  <!-- See also: spec bug 7624 -->

  <?err XTTE0945?>
  
<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="a">
  <xsl:value-of select="local:value(.)"/>
</xsl:template>

<xsl:function name="local:value">
  <xsl:param name="arg"/>
  <xsl:copy>
    <t>23</t>
  </xsl:copy>
</xsl:function>

</xsl:stylesheet>