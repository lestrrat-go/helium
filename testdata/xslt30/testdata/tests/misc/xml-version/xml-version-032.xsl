<?xml version="1.1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- PURPOSE: See if a namespace undeclaration (using undeclare-prefixes=" 1 ") in a stylesheet works -->

<xsl:output method="xml" version="1.1" undeclare-prefixes=" 1 "/>

<xsl:template match="/">
  <doc xmlns:a="http://a/" xsl:inherit-namespaces="no">
    <chap xsl:inherit-namespaces="no">
      <para/>
      <para xmlns:a=""/>
    </chap>
  </doc>
</xsl:template>
 
</xsl:stylesheet>
