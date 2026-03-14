<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- test inherit-namespaces="no" on literal result elements -->


<xsl:output method="xml" version="1.1"  undeclare-prefixes="yes"/>

<xsl:template match = "/">
  <a xmlns:n="http://n/" xsl:inherit-namespaces="no">
    <xsl:call-template name="c"/>
  </a>
</xsl:template>

<xsl:template name="c">
  <b/>
</xsl:template>

</xsl:stylesheet>
