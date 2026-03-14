<t:stylesheet xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <t:import href="next-match-019d.xsl"/>
  
  <t:template match="bag">
    <t:param name="p1" select="'default'"/>
    <imp-b><t:value-of select="$p1"/></imp-b>
  </t:template>
  
  <t:template match="tag">
    <t:param name="p1" select="'default'"/>
    <imp1-t><t:value-of select="$p1"/></imp1-t>
    <t:text>&#10;</t:text>
    <t:apply-imports>
      <t:with-param name="p1" select="'imported template 3'"/>
    </t:apply-imports>
  </t:template>
  
  
</t:stylesheet>
