<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns="http://www.w3.org/1999/xhtml"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0">

   <t:output method="xhtml"
             cdata-section-elements="example"
             indent="no"
             encoding="US-ASCII"
             use-character-maps="format1" />
   <!-- Purpose: Test XHTML output with cdata-section-elements that have non-ASCII character -->
   
   <t:character-map name="format1">
      <t:output-character character="&#170;" string="A"/>
   </t:character-map>

   <t:template match="/">
      <html>
         <out>
            <example>foo &#xaa; bar</example>
         </out>
      </html>
   </t:template>
</t:transform>
