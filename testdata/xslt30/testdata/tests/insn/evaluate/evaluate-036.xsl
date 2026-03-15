<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:e36="http://evaluate-036/" exclude-result-prefixes="xs e36">

  <!-- PURPOSE test xsl:evaluate with schema-aware="1" -->

  <xsl:import-schema namespace="http://evaluate-036/">
    <xs:schema targetNamespace="http://evaluate-036/">
      <xs:simpleType name="calendar">
        <xs:union memberTypes="xs:date xs:time xs:dateTime"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:template name="main">
    <out>
      <xsl:evaluate schema-aware="1" xpath="'current-date() instance of e36:calendar'"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
