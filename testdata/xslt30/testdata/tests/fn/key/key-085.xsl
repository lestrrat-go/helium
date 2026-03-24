<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: key dependent on value of a parameter. -->

  <xsl:output method="xml" encoding="UTF-8" indent="no"/>
  
  <xsl:param name="min" required="yes"/>

  <xsl:key name="places" match="town[string-length(@name) gt $min]" use="@state"/>

  <xsl:template match="doc">
    <out>
      <xsl:value-of select="key('places', 'MA')/@name"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
