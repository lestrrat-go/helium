<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-round-half-to-even?>
<!-- PURPOSE:  Test of round-half-to-even on integers. -->
    <!-- variant of 3301 forcing run-time evaluation -->
    
    <xsl:param name="p" select="1"/>

<xsl:template match="doc">
<out>
    <a b="{round-half-to-even($p*120)}" c="{round-half-to-even($p*0)}" d="{round-half-to-even($p*-3)}"/>
    <a b="{round-half-to-even($p*120, 2)}" c="{round-half-to-even($p*0, 2)}" d="{round-half-to-even($p*-3, 2)}"/>
    <a b="{round-half-to-even($p*120, -2)}" c="{round-half-to-even($p*0, -2)}" d="{round-half-to-even($p*-3, -2)}"/>
    <a b="{round-half-to-even($p*180, -2)}" c="{round-half-to-even($p*150, -2)}" d="{round-half-to-even($p*250, -2)}"/>
    <a b="{round-half-to-even($p*-180, -2)}" c="{round-half-to-even($p*-150, -2)}" d="{round-half-to-even($p*-250, -2)}"/>
</out>
</xsl:template>
</xsl:stylesheet>