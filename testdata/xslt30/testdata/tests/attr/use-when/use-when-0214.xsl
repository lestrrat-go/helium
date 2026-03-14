<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
      <!-- Purpose: Test case in which use-when is used with element-available and function-available functions
  				Where the element and function do NOT exist.  -->

      <t:template match="/">
            <out>
                  <t:element name="E1" use-when="element-available('t:values-of')">
                        <t:text>Element defined</t:text>
                  </t:element>
                  <E2 t:use-when="function-available('documents')">
                        <t:text>function IS defined</t:text>
                  </E2>
            </out>
      </t:template>
</t:transform>
