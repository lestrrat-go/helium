<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
  <!-- Purpose: Test case that uses use-when on xsl:element where the EBV of the attribute results to false.  -->

  <t:template match="doc">
    <out>
      <t:element name="e">
        <t:element name="e1" use-when="1=2">
          <t:element name="e1.1">v1.1</t:element>
          <t:element name="e1.2" use-when="1=1">v1.2 <t:element name="e1.2.1">v1.2.1</t:element>
          </t:element>
        </t:element>
      </t:element>
    </out>
  </t:template>
</t:transform>
