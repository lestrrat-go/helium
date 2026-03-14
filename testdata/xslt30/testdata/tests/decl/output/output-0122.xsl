<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">
<!-- Purpose: All xsl:output elements are merged into a single element. While repeats of
    most attributes are just tested for conflicts, cdata-section-elements will contain the
    union of the specified values. Both example and test should be wrapped by CDATA, and
    the output should be XML (since cdata-section-elements only applies to XML).-->

   <t:output cdata-section-elements="test" encoding="UTF-8" indent="no"/>

   <t:output method="xml" cdata-section-elements="example"/>

   <t:template match="/">
      <out>
         <example>&lt;foo&gt;</example>
         <plain>bar &amp; ban</plain>
         <test>!&gt;</test>
      </out>
   </t:template>
</t:transform>
