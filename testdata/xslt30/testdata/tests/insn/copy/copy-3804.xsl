<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- Test atomizing works after deep copy -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>  

<xsl:template match="/">
  <xsl:variable name="temp">
    <z1>
      <z2>
        <z3/>
        <xsl:copy-of select="."/>
        <z4/>
        <xsl:copy-of select="./*"/>
      </z2>
    </z1>
  </xsl:variable>
  <out>
    <c><xsl:value-of select="contains($temp//z2/node()[last()], 'Success')"/></c>
  </out>
</xsl:template>

</xsl:stylesheet>