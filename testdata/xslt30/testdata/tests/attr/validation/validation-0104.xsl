<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             default-validation="strict">
<!-- Purpose: Test that an error is raised if @default-validation=strict is present in 
  				the stylesheet but the processor is a basic processor.-->

   <t:template match="/doc">
      <t:element name="elem">2.2</t:element>
   </t:template>
</t:transform>
