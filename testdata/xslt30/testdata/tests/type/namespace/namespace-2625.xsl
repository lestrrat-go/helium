<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- test inherit-namespaces="false" on xsl:copy -->


<xsl:output method="xml" version="1.1" undeclare-prefixes="yes"/>

<xsl:template match = "/">
  <xsl:variable name="e" as="element()">
    <n:a xmlns:n="http://n/" xmlns="http://d/"/>
  </xsl:variable>
  <xsl:for-each select="$e">
    <xsl:copy inherit-namespaces="false">
      <xsl:call-template name="c"/>
    </xsl:copy>
  </xsl:for-each>
</xsl:template>

<xsl:template name="c">
  <n:b xmlns:n="http://n2/"/>
  <c xmlns="http://n3/"/>  
</xsl:template>

</xsl:stylesheet>
