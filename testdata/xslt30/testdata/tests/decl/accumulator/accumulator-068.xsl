<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  xmlns:math="http://www.w3.org/2005/xpath-functions/math"
  xmlns:map="http://www.w3.org/2005/xpath-functions/map" exclude-result-prefixes="xs math map"
  version="3.0">

  <xsl:param name="STREAMABLE" static="yes" as="xs:boolean" select="true()"/>

  <xsl:mode _streamable="{$STREAMABLE}" on-no-match="shallow-skip"
    use-accumulators="notification-id map-notifid-msg"/>

  <xsl:param name="notif-uri" as="xs:string" select="'accumulator-068b.xml'"/>

  <xsl:accumulator name="notification-id" as="xs:integer?" initial-value="()"
    _streamable="{$STREAMABLE}">
    <xsl:accumulator-rule match="Notification" select="()"/>
    <xsl:accumulator-rule match="Notification/NotifId/text()" select="xs:integer(.)"/>
  </xsl:accumulator>

  <xsl:accumulator name="map-notifid-msg" as="map(xs:integer, xs:string)" initial-value="map{}"
    _streamable="{$STREAMABLE}">
    <xsl:accumulator-rule match="Notification/MsgText/text()"
      select="map:put($value, accumulator-before('notification-id'), string())"/>
  </xsl:accumulator>

  <xsl:output method="adaptive"/>

  <xsl:template name="xsl:initial-template">
    <out>
      <xsl:variable name="result-map" as="map(*)">
        <xsl:source-document href="{$notif-uri}" _streamable="{$STREAMABLE}"
          use-accumulators="notification-id map-notifid-msg">
          <xsl:apply-templates/>
          <xsl:sequence select="accumulator-after('map-notifid-msg')"/>
        </xsl:source-document>
      </xsl:variable>
      <xsl:value-of select="serialize($result-map, map{'method':'json'})"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
