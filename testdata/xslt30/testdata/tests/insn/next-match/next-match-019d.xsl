<t:stylesheet xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <t:template match="tag">
    <t:param name="p1" select="'default'"/>
    <imp1-t>
      <t:value-of select="$p1"/>
    </imp1-t>
    <t:text>&#10;</t:text>
  </t:template>

</t:stylesheet>
