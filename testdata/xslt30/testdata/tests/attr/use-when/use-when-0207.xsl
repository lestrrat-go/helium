<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- Purpose: Test case that uses use-when on xsl:attribute where the EBV of the attribute results to false.  -->

  <t:template match="doc">
    <out>
      <t:element name="book">
        <t:attribute name="chapters" select="1 to 5" use-when="false()"/>
      </t:element>
    </out>
  </t:template>
</t:transform>
