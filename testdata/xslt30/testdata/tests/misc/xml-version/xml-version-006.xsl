<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase that uses LSEP(0x2028) character as actual value in the stylesheet document.
  				The character should create a line feed in the output document.  -->

   <t:output method="xml" version="1.1" encoding="UTF-8"/>

   <t:template match="/">
      <out xml:space="preserve">
</out>
  </t:template>
</t:transform>
