<?xml version="1.1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- PURPOSE: Test AVT in @undeclare-prefixes of xsl:result-document, which evaluates to " yes ". -->

<xsl:param name="yes"> yes </xsl:param>

<xsl:template match="/">
  <xsl:result-document method="xml" output-version="1.1" undeclare-prefixes="{$yes}">
  <doc xmlns:a="http://a/" xsl:inherit-namespaces="no">
    <chap xsl:inherit-namespaces="no">
      <para/>
      <para xmlns:a=""/>
    </chap>
  </doc>
  </xsl:result-document>
</xsl:template>
 
</xsl:stylesheet>
