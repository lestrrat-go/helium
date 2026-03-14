<?xml version="1.1"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec serialization#xml-undeclare-prefixes?>
  <!-- PURPOSE: Representation of namespace undeclaration with undeclare="no" -->

<xsl:output method="xml" version="1.1" undeclare-prefixes="no"/>

<xsl:template match="/">
  <doc xmlns:a="http://a/">
    <chap xsl:inherit-namespaces="no">
      <para/>
      <para xmlns:a=""/>
    </chap>
  </doc>
</xsl:template>
 
</xsl:stylesheet>
