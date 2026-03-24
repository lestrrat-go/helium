<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests #x13F character in the name of an xsl:element when the XML version of the stylesheet is 1.1.  -->

   <t:output method="xml" version="1.1" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:element name="testğ"/>
      </out>
  </t:template>
</t:transform>
