<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" 
  xmlns:e34="http://evaluate-034/" exclude-result-prefixes="xs e34">

  <!-- PURPOSE test xsl:evaluate with schema-aware="true" -->

  <xsl:import-schema namespace="http://evaluate-034/">
    <xs:schema targetNamespace="http://evaluate-034/">
      <xs:simpleType name="calendar">
        <xs:union memberTypes="xs:date xs:time xs:dateTime"/>
      </xs:simpleType>
    </xs:schema>
  </xsl:import-schema>
  
  <xsl:template name="main">
    <out>
      <xsl:evaluate schema-aware="true" xpath="'current-date() instance of e34:calendar'"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
