<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" default-validation="strip">

  <!-- PURPOSE: error case, target namesapce does not match. -->
 
   <xsl:import-schema schema-location="schema075.xsd" namespace="http://this-is-not-the-right-namespace/"/>
 
  <xsl:template match="/">
    <out/>
  </xsl:template>


</xsl:stylesheet>
