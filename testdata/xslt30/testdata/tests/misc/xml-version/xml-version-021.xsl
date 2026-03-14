<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase that uses LSEP(0x2028) character in xsl:strip-space instruction.
  				The character reference should appear in the output document.  -->

   <t:output method="xml" version="1.1"/>

   <t:strip-space elements="Country"/>

   <t:template match="@*|node()">
      <t:copy>
         <t:apply-templates select="@*|node()"/>
      </t:copy>
  </t:template>
</t:transform>
