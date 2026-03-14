<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-round-half-to-even?>
<!-- PURPOSE:  Test of round-half-to-even on integers. -->

<xsl:template match="doc">
<out>
<a b="{round-half-to-even(120)}" c="{round-half-to-even(0)}" d="{round-half-to-even(-3)}"/>
<a b="{round-half-to-even(120, 2)}" c="{round-half-to-even(0, 2)}" d="{round-half-to-even(-3, 2)}"/>
<a b="{round-half-to-even(120, -2)}" c="{round-half-to-even(0, -2)}" d="{round-half-to-even(-3, -2)}"/>
<a b="{round-half-to-even(180, -2)}" c="{round-half-to-even(150, -2)}" d="{round-half-to-even(250, -2)}"/>
<a b="{round-half-to-even(-180, -2)}" c="{round-half-to-even(-150, -2)}" d="{round-half-to-even(-250, -2)}"/>
</out>
</xsl:template>
</xsl:stylesheet>