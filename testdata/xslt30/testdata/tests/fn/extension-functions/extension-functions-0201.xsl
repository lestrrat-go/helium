<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:math="http://exslt.org/math"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="1.0"
             extension-element-prefixes="math">
<!-- Purpose: Test case to verify that a dynamic error is raised if the arguments supplied 
  				to a call on an extension function do not satisfy the rules defined for that 
  				particular extension function -->

   <t:variable name="cstant1" select="&#34;PI&#34;"/>

   <t:variable name="precision2" select="1"/>

   <t:template match="/">
      <group1 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
      PI:
      <t:value-of select="math:constant($precision2,$cstant1)"/>
      </group1>
   </t:template>
</t:transform>
