<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

   <!-- Tests a sizeable node-valued static variable -->
   
   <xsl:variable name="tree" select="parse-xml('&lt;doc>' || string-join((1 to 1000)!('&lt;e>' || . || '&lt;/e>')) || '&lt;/doc>')" static="true"/>
   
   <xsl:template name="xsl:initial-template">
      <result count="{count($tree//e)}"/>
   </xsl:template>
   
</xsl:transform>
