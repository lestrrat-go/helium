<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses IRI on the name of the element, as well as its namespace  -->

   <t:output method="xml" version="1.1" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <t:element name="pre:outÃ" namespace="http://example.org/preÀ"/>
      </out>
  </t:template>
</t:transform>
