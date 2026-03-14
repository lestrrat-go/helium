<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:co="http://www.example.com"
             xmlns:ed="http://www.example.edu"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             exclude-result-prefixes="co ed">
<!-- Purpose: Test that pattern "*:NCName" gets a default priority of -0.25. -->

   <t:output method="xml" indent="no" encoding="UTF-8"/>

   <t:template match="doc">
      <out>
         <t:apply-templates select="*"/>
      </out>
   </t:template>

   <t:template match="ed:b" priority="-0.1">
      <t:text>,Fully-qualified-1:</t:text>
      <t:value-of select="@mark"/>
   </t:template>

   <t:template match="*:b">
      <t:text>,Localnamed-b:</t:text>
      <t:value-of select="@mark"/>
   </t:template>

   <t:template match="co:b" priority="-0.4">
      <t:text>,Fully-qualified-4:</t:text>
      <t:value-of select="@mark"/>
   </t:template>
</t:transform>
