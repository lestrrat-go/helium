<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:e38="http://evaluate-038/" exclude-result-prefixes="xs e38">

  <!-- PURPOSE test xsl:evaluate with schema-aware="TRUE" (not allowed) -->

  <xsl:import-schema namespace="http://evaluate-038/">
    <xs:schema targetNamespace="http://evaluate-038/">
      <xs:simpleType name="calendar">
        <xs:union memberTypes="xs:date xs:time xs:dateTime"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:template name="main">
    <out>
      <xsl:evaluate schema-aware="TRUE" xpath="'current-date() instance of e38:calendar'"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
