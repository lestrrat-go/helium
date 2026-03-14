<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test doc() function with path following.-->

   <t:template match="catalog">
      <out>
         <t:apply-templates select="doc(pointer/urlref/@urlstr)/market.participant/business.identity.group/business.name"/>
         <t:apply-templates select="doc('doc07.xml')/market.participant/address.set/*"/>
      </out>
   </t:template>

   <t:template match="location.in.street">
      <t:apply-templates/>
   </t:template>

   <t:template match="street">
      <t:apply-templates/>
      <t:text/>
   </t:template>

   <t:template match="city">
      <t:apply-templates/>
      <t:text>, </t:text>
   </t:template>

   <t:template match="country.subentity">
      <t:apply-templates/>
      <t:text> </t:text>
   </t:template>

   <t:template match="postcode">
      <t:apply-templates/>
   </t:template>

   <t:template match="telephone.number">
      <t:apply-templates/>
   </t:template>
</t:transform>
