<?xml version="1.1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- PURPOSE: See if a namespace undeclaration (using undeclare-prefixes="yes" on xsl:result-document) in a stylesheet works -->

<xsl:template match="/">
  <xsl:result-document method="xml" output-version="1.1" undeclare-prefixes="yes">
  <doc xmlns:a="http://a/" xsl:inherit-namespaces="no">
    <chap xsl:inherit-namespaces="no">
      <para/>
      <para xmlns:a=""/>
    </chap>
  </doc>
  </xsl:result-document>
</xsl:template>
 
</xsl:stylesheet>
